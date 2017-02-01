# worktimer-gtk

Status Icon timer with json output.

Decodes saved json and adds up the hours.

### Debian Installation

You need to have Go installed to compile worktimer-gtk

```
sudo apt-get install libgtk2.0-dev pango1.0-dev libgdk-pixbuf2.0-dev 
go get -v -u github.com/aerth/worktimer-gtk
sudo mv $GOPATH/bin/worktimer-gtk /usr/local/bin/
```
