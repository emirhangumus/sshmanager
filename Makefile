install: check_sshpass
	go build . && cp sshmanager ~/.local/bin

remove:
	rm ~/.local/bin/sshmanager

clean:
	rm sshmanager

check_sshpass:
	@command -v sshpass >/dev/null 2>&1 && \
	echo '`sshpass` is already installed.' || \
	(sudo apt install sshpass -y && \
	echo '`sshpass` is installed.')