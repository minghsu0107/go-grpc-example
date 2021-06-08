# Calculator Service
## Usage
In the greering service example, we have learned how to build client and server from source and run them locally. Now, let's containerize the calculator service:

Build server container:
```bash
./build.sh server
```
Build client container:
```bash
./build.sh client
```
Run server:
```bash
docker run --rm --network host server:v1
```
Run client:
```bash
docker run --rm --network host client:v1
```