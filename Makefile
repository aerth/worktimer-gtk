all:
	go build --ldflags='-s' -o worktimer-gtk
install:
	@mv worktimer-gtk /usr/local/bin