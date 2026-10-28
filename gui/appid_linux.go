//go:build linux

package main

/*
#cgo pkg-config: gio-2.0
#include <stdlib.h>
#include <gio/gio.h>

static void set_app_id(const char *id) {
	GApplication *app = g_application_get_default();
	if (app != NULL && !g_application_get_is_registered(app)) {
		g_application_set_application_id(app, id);
	}
}
*/
import "C"

import "unsafe"

// AppID é o identificador do app no Linux: o mesmo do .desktop, do AppStream
// e do Flatpak.
const AppID = "io.github.PauloTorresApps.gopro_gps_repair"

// setLinuxAppID troca o identificador que o Wails dá à aplicação GTK
// ("org.wails.<nome>") por AppID. O Wails cria a aplicação GTK dentro de
// app.Run; por isso a chamada fica em RepairService.ServiceStartup, que roda
// logo depois disso e antes de a aplicação ser registrada.
//
// Sem isso, dentro do Flatpak o WebKit fecha o app: o nome que ele registra
// no barramento de acessibilidade tem que começar pelo ID do Flatpak. Fora do
// Flatpak, o ID certo faz o GNOME/KDE associarem a janela ao ícone e ao
// atalho do menu.
func setLinuxAppID() {
	id := C.CString(AppID)
	defer C.free(unsafe.Pointer(id))
	C.set_app_id(id)
}
