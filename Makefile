TAG=latest

deploy:
	echo "Building image..."
	docker build -f ./Dockerfile -t dekuyo/gopray:${TAG} --target=production --no-cache .
	echo "Deploying image to Docker Hub..."
	docker image push dekuyo/gopray:${TAG}

test:
	go test -coverprofile=coverage.txt  -covermode=count ./...

.PHONY: deploy test
