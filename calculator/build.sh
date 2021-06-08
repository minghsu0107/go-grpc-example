IMAGE_TAG="v1"
SVC_NAME=$1
docker build . -t "$SVC_NAME:$IMAGE_TAG" --build-arg SVC_NAME=$SVC_NAME