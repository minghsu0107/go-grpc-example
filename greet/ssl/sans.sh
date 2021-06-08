# server.key: Server private key
# server.crt: Server certificate (public key) signed using SANS (one certificate for multiple domains)
# CN: Common Name - must be the same as the Web address you will be accessing when connecting to a secure site
# we use SANS here because go 1.15+ deprecates Common Name Field

openssl genrsa -out prikey.pem 2048 # or 4096
openssl req -nodes -new -x509 -sha256 -days 3650 -config cert.conf -extensions 'req_ext' -key prikey.pem -out cert.pem