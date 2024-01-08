TAG=latest

deploy:
	echo "Building image..."
	docker build -f ./Dockerfile -t dekuyo/gopray:${TAG} --target=production --no-cache .
	echo "Deploying image to Docker Hub..."
	docker image push dekuyo/gopray:${TAG}

test:
	go test -cover -coverprofile=coverage.txt  -covermode=count ./... | gocol 

.PHONY: deploy test
