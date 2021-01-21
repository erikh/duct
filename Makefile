docker-image:
	docker build -t quay.io/erikh/duct-test .

push: docker-image
	docker push quay.io/erikh/duct-test

test: docker-image
	docker run -v ${PWD}:/code -w /code --privileged quay.io/erikh/duct-test sh test.sh
