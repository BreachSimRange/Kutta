BIN_NAME = kutta
INSTALL_DIR = /usr/local/bin
SERVICE_FILE = /etc/systemd/system/$(BIN_NAME).service
SERVICE_USER = www-data
SERVICE_GROUP = www-data
STATE_DIR = /var/kutta

build:
	go build -o $(BIN_NAME)

install:
	sudo mv $(BIN_NAME) $(INSTALL_DIR)/

service:
	sudo mkdir -p $(STATE_DIR)
	sudo chown -R $(SERVICE_USER):$(SERVICE_GROUP) $(STATE_DIR)
	sudo cp packaging/$(BIN_NAME).service $(SERVICE_FILE)
	sudo systemctl daemon-reload
	sudo systemctl enable $(BIN_NAME)
	sudo systemctl restart $(BIN_NAME)

uninstall:
	sudo systemctl stop $(BIN_NAME) || true
	sudo systemctl disable $(BIN_NAME) || true
	sudo rm -f $(SERVICE_FILE)
	sudo rm -f $(INSTALL_DIR)/$(BIN_NAME)
	sudo systemctl daemon-reload
