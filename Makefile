generate:
	mkdir -p api/thumbnail_v1
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		api/thumbnail_v1/thumbnail.proto

clean_generate:
	rm -rf api/thumbnail_v1

clean_db:
	rm -f thumbnail.db

test:
	go test -v ./...