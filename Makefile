CFLAGS=-g
export CFLAGS
deploy:
	echo "Building image..."
	docker build -f ./Dockerfile -t dekuyo/gopray:${TAG} --target=production --no-cache .
	echo "\nDeploying image to Docker Hub..."
	docker image push dekuyo/gopray:${TAG}

.PHONY: deploy