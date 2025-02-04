---

## Differences from original repo

- Fixed feh flag
- ...

---

# Wallpaper

A cross-platform (Linux, Windows, and macOS) Golang library for getting and setting the desktop background.

## Installation

```sh
go get github.com/ktkv419/wallpaper
```

## Example

```go
package main

import (
	"fmt"

	"github.com/reujab/wallpaper"
)

func main() {
	background, err := wallpaper.Get()
	check(err)
	fmt.Println("Current wallpaper:", background)

	err = wallpaper.SetFromFile("/usr/share/backgrounds/gnome/adwaita-day.jpg")
	check(err)

	err = wallpaper.SetFromURL("https://i.imgur.com/pIwrYeM.jpg")
	check(err)

	err = wallpaper.SetMode(wallpaper.Crop)
	check(err)
}

```

## Supported desktops

* Windows
* macOS
* GNOME
* KDE
* Cinnamon
* Unity
* Budgie
* XFCE
* LXDE
* MATE
* Deepin
* Most Wayland compositors (set only, requires swaybg)
* i3 (set only, requires feh)
