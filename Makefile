generate:
		go run main.go $(VERSION)

install_deps:
		go mod vendor