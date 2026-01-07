BIN_NAME = kutta
INSTALL_DIR = /usr/local/bin
SERVICE_FILE = /etc/systemd/system/$(BIN_NAME).service

build:
	go build -o $(BIN_NAME)

install:
	sudo mv $(BIN_NAME) $(INSTALL_DIR)/

service:
	sudo mkdir /var/kutta
	sudo cp packaging/$(BIN_NAME).service $(SERVICE_FILE)
	sudo systemctl daemon-reload
	sudo systemctl enable $(BIN_NAME)
	sudo systemctl start $(BIN_NAME)

uninstall:
	sudo systemctl stop $(BIN_NAME) || true
	sudo systemctl disable $(BIN_NAME) || true
	sudo rm -f $(SERVICE_FILE)
	sudo rm -f $(INSTALL_DIR)/$(BIN_NAME)
	sudo systemctl daemon-reload
