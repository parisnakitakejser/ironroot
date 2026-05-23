from http.server import SimpleHTTPRequestHandler, HTTPServer
import ssl

httpd = HTTPServer(("0.0.0.0", 8444), SimpleHTTPRequestHandler)
context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
context.load_cert_chain(certfile="../../certs/tls.crt", keyfile="../../certs/tls.key")
httpd.socket = context.wrap_socket(httpd.socket, server_side=True)
print("Serving https://demo.home.arpa:8444")
httpd.serve_forever()
