# Proto генерируется из корня Backend: cd .. && make proto
.PHONY: proto
proto:
	$(MAKE) -C .. proto

.PHONY: swagger
swagger:
	swag init -g cmd/main.go -o docs --parseDependency --parseInternal
