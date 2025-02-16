install: check_sshpass build
	cp sshmanager ~/.local/bin

install_compressed: check_sshpass check_upx build_compressed
	cp sshmanager ~/.local/bin

build:
	go build -ldflags="-s -w" .

build_compressed: check_upx build
	upx --best --lzma sshmanager

remove:
	rm ~/.local/bin/sshmanager

clean:
	rm -f sshmanager

check_sshpass:
	@command -v sshpass >/dev/null 2>&1 && \
	echo '`sshpass` is already installed.' || \
	(sudo apt install sshpass -y && \
	echo '`sshpass` is installed.')

check_upx:
	@command -v upx >/dev/null 2>&1 && \
	echo '`upx` is already installed.' || \
	(sudo apt install upx -y && \
	echo '`upx` is installed.')
