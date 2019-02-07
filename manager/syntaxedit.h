/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

#ifndef SYNTAXEDIT_H
#define SYNTAXEDIT_H

#include <stdbool.h>
#include <windows.h>
#include <richedit.h>

#define SYNTAXEDIT_STYLE (WS_CHILD | WS_CLIPSIBLINGS | ES_MULTILINE | WS_VISIBLE | WS_VSCROLL | WS_HSCROLL | WS_BORDER | WS_TABSTOP | ES_WANTRETURN | ES_NOOLEDRAGDROP)
#define SYNTAXEDIT_EXTSTYLE (WS_EX_CLIENTEDGE)

/* The old MFC reflection trick. */
#define WM_REFLECT (WM_USER + 0x1C00)

#define SE_PRIVATE_KEY (WM_USER + 0x3100)

extern bool register_syntax_edit(void);

#endif
