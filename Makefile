install:
	go build . && cp sshmanager ~/.local/bin

remove:
	rm ~/.local/bin/sshmanager

clean:
	rm sshmanager