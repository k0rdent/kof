#!/usr/bin/env bash
set -euo pipefail

DIR="${1:-../dev/ssl}"

mkdir -p ${DIR}

DIR=$(realpath ${DIR})

cat << EOF > ${DIR}/req.cnf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = dex.example.com
EOF

openssl genrsa -out ${DIR}/ca-key.pem 2048
openssl req -x509 -new -nodes -key ${DIR}/ca-key.pem -days 365 -out ${DIR}/ca.pem -subj "/CN=kube-ca"

openssl genrsa -out ${DIR}/key.pem 2048
openssl req -new -key ${DIR}/key.pem -out ${DIR}/csr.pem -subj "/CN=kube-ca" -config ${DIR}/req.cnf
openssl x509 -req -in ${DIR}/csr.pem -CA ${DIR}/ca.pem -CAkey ${DIR}/ca-key.pem -CAcreateserial -out ${DIR}/cert.pem -days 10 -extensions v3_req -extfile ${DIR}/req.cnf

echo "✅ Certificates have been generated to the ${DIR}"

CERT_B64=$(cat ${DIR}/cert.pem | base64 -w 0)
KEY_B64=$(cat ${DIR}/key.pem | base64 -w 0)

echo "🔑 Creating Secret with CA from ${DIR}/ca.pem..."

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: dex-tls
  namespace: kof
type: kubernetes.io/tls
data:
  tls.crt: $CERT_B64
  tls.key: $KEY_B64
EOF
