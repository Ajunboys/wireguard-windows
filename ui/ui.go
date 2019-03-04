/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/windows/conf"
	"golang.zx2c4.com/wireguard/windows/service"
	"golang.zx2c4.com/wireguard/windows/ui/syntax"
)

const demoConfig = `[Interface]
PrivateKey = 6KpcbNFK4tKBciKBT2Rj6Z/sHBqxdV+p+nuNA5AlWGI=
Address = 192.168.4.84/24
DNS = 8.8.8.8, 8.8.4.4, 1.1.1.1, 1.0.0.1

[Peer]
PublicKey = JRI8Xc0zKP9kXk8qP84NdUQA04h6DLfFbwJn4g+/PFs=
Endpoint = demo.wireguard.com:12912
AllowedIPs = 0.0.0.0/0
`

func RunUI() {
	icon, _ := walk.NewIconFromResourceId(8)

	mw, _ := walk.NewMainWindowWithName("WireGuard")
	tray, _ := walk.NewNotifyIcon(mw)
	defer tray.Dispose()
	tray.SetIcon(icon)
	tray.SetToolTip("WireGuard: Deactivated")
	tray.SetVisible(true)

	mw.SetSize(walk.Size{900, 800})
	mw.SetLayout(walk.NewVBoxLayout())
	mw.SetIcon(icon)
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		mw.Hide()
	})

	tl, _ := walk.NewTextLabel(mw)
	tl.SetText("Public key: (unknown)")

	se, _ := syntax.NewSyntaxEdit(mw)
	lastPrivate := ""
	se.PrivateKeyChanged().Attach(func(privateKey string) {
		if privateKey == lastPrivate {
			return
		}
		lastPrivate = privateKey
		key := func() string {
			if privateKey == "" {
				return ""
			}
			decoded, err := base64.StdEncoding.DecodeString(privateKey)
			if err != nil {
				return ""
			}
			if len(decoded) != 32 {
				return ""
			}
			var p [32]byte
			var s [32]byte
			copy(s[:], decoded[:32])
			curve25519.ScalarBaseMult(&p, &s)
			return base64.StdEncoding.EncodeToString(p[:])
		}()
		if key != "" {
			tl.SetText("Public key: " + key)
		} else {
			tl.SetText("Public key: (unknown)")
		}
	})
	se.SetText(demoConfig)

	pb, _ := walk.NewPushButton(mw)
	pb.SetText("Start")
	var runningTunnel *service.Tunnel
	pb.Clicked().Attach(func() {
		restoreState := true
		pbE := pb.Enabled()
		seE := se.Enabled()
		pbT := pb.Text()
		defer func() {
			if restoreState {
				pb.SetEnabled(pbE)
				se.SetEnabled(seE)
				pb.SetText(pbT)
			}
		}()
		pb.SetEnabled(false)
		se.SetEnabled(false)
		pb.SetText("Requesting..")
		if runningTunnel != nil {
			err := runningTunnel.Stop()
			if err != nil {
				walk.MsgBox(mw, "Unable to stop tunnel", err.Error(), walk.MsgBoxIconError)
				return
			}
			restoreState = false
			runningTunnel = nil
			return
		}
		c, err := conf.FromWgQuick(se.Text(), "test")
		if err != nil {
			walk.MsgBox(mw, "Invalid configuration", err.Error(), walk.MsgBoxIconError)
			return
		}
		tunnel, err := service.IPCClientNewTunnel(c)
		if err != nil {
			walk.MsgBox(mw, "Unable to create tunnel", err.Error(), walk.MsgBoxIconError)
			return
		}
		err = tunnel.Start()
		if err != nil {
			walk.MsgBox(mw, "Unable to start tunnel", err.Error(), walk.MsgBoxIconError)
			return
		}
		restoreState = false
		runningTunnel = &tunnel
	})

	quitAction := walk.NewAction()
	quitAction.SetText("Exit")
	quitAction.Triggered().Attach(func() {
		tray.Dispose()
		_, err := service.IPCClientQuit(true)
		if err != nil {
			walk.MsgBox(nil, "Error Exiting WireGuard", fmt.Sprintf("Unable to exit service due to: %s. You may want to stop WireGuard from the service manager.", err), walk.MsgBoxIconError)
			os.Exit(1)
		}
	})
	tray.ContextMenu().Actions().Add(quitAction)
	tray.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			mw.Show()
			win.SetForegroundWindow(mw.Handle())
		}
	})

	setServiceState := func(tunnel *service.Tunnel, state service.TunnelState, showNotifications bool) {
		if tunnel.Name != "test" {
			return
		}
		//TODO: also set tray icon to reflect state
		switch state {
		case service.TunnelStarting:
			se.SetEnabled(false)
			pb.SetText("Starting...")
			pb.SetEnabled(false)
			tray.SetToolTip("WireGuard: Activating...")
		case service.TunnelStarted:
			se.SetEnabled(false)
			pb.SetText("Stop")
			pb.SetEnabled(true)
			tray.SetToolTip("WireGuard: Activated")
			if showNotifications {
				//TODO: ShowCustom with right icon
				tray.ShowInfo("WireGuard Activated", fmt.Sprintf("The %s tunnel has been activated.", tunnel.Name))
			}
		case service.TunnelStopping:
			se.SetEnabled(false)
			pb.SetText("Stopping...")
			pb.SetEnabled(false)
			tray.SetToolTip("WireGuard: Deactivating...")
		case service.TunnelStopped, service.TunnelDeleting:
			if runningTunnel != nil {
				runningTunnel.Delete()
				runningTunnel = nil
			}
			se.SetEnabled(true)
			pb.SetText("Start")
			pb.SetEnabled(true)
			tray.SetToolTip("WireGuard: Deactivated")
			if showNotifications {
				//TODO: ShowCustom with right icon
				tray.ShowInfo("WireGuard Deactivated", fmt.Sprintf("The %s tunnel has been deactivated.", tunnel.Name))
			}
		}
	}
	service.IPCClientRegisterTunnelChange(func(tunnel *service.Tunnel, state service.TunnelState) {
		setServiceState(tunnel, state, true)
	})
	go func() {
		tunnels, err := service.IPCClientTunnels()
		if err != nil {
			return
		}
		for _, tunnel := range tunnels {
			state, err := tunnel.State()
			if err != nil {
				continue
			}
			runningTunnel = &tunnel
			setServiceState(&tunnel, state, false)
		}
	}()

	mw.Run()
}
