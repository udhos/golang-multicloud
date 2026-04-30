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

### Backend on azure

- Create an Azure CosmosDB account with the MongoDB API.
- Azure will provide you with a connection string that you can use to connect to the CosmosDB account.
- Set the MONGO_SERVER_URL environment variable to your CosmosDB connection string (the mongodocstore driver reads this env var).

```bash
export MONGO_SERVER_URL="mongodb+srv://USER:PASSWORD@ACCOUNT.global.mongocluster.cosmos.azure.com/?tls=true&authMechanism=SCRAM-SHA-256&retrywrites=false&maxIdleTimeMS=120000"

shopping-cart-backend -config config-examples/azure.yaml
```

### Backend on gcp

- Enable Firestore API in your GCP project.
- Set GOOGLE_APPLICATION_CREDENTIALS to your service account key file,
    or use Application Default Credentials (gcloud auth application-default login).
- Edit config-examples/gcp.yaml and replace YOUR_GCP_PROJECT_ID with your actual project ID.

```bash
shopping-cart-backend -config config-examples/gcp.yaml
```

## Frontend - Embedded on backend

For convenience, the frontend static files are embedded in the backend binary.

Just open your browser at http://localhost:8080/ to see the shopping cart frontend.

## Frontend - Standalone

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
