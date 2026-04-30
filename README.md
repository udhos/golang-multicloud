# golang-multicloud

[golang-multicloud](https://github.com/udhos/golang-multicloud) is a demo using google/go-cloud Go CDK to illustrate how to build a multi-cloud ready application.

# Build

```bash
./build.sh
```

# Run

## Backend

Run the shopping cart backend.

### Backend on local volatile memory

```bash
shopping-cart-backend
```

### Backend on aws

Create a DynamoDB table named "shopping-carts" with partition key "ID" (String).

```bash
shopping-cart-backend -config config-examples/aws.yaml
```

## Frontend

Run the shopping cart frontend.

You will need to serve the frontend js application.

You can use any static file server.

Do not run the file server on port 8080 because the backend by default runs on that port.

For example:

```bash
# using gowebhello

cd shopping-cart-frontend

go run github.com/udhos/gowebhello/gowebhello@latest -addr :8000
```

Then open your browser at http://localhost:8000/www/ to see the shopping cart frontend.

```bash
# using python

cd shopping-cart-frontend

python3 -m http.server 8000
```

Then open your browser at http://localhost:8000 to see the shopping cart frontend.
