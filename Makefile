APP_NAME=job-connect

# generate swagger docs
swag:
	swag init -g main.go --parseDependency --parseInternal

# run app
run:
	go run main.go

# run everything (swagger + server)
dev: swag run

# JUST RUN WITH THIS COMMANDS FROM TERMINAL
# make dev