// nexus-mini — энэ репогийн өөрийн дистрибуц: цөм + backend/apps доторх
// модулиуд. Гадны компани өөрийн дистрибуцдаа яг ийм main бичнэ
// (docs/03-module-guide.md «Өөрийн дистрибуц»).
package main

import (
	"github.com/gerege-systems/nexus-mini/backend/apps"
	"github.com/gerege-systems/nexus-mini/backend/core"
)

func main() { core.Main(apps.All()...) }
