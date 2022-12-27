```bash
docker build -f api.Dockerfile -t services/api:$TAG .

# Only on local and if using kind
kind load docker-image services/api:$TAG

kubectl apply -f api-deployment.yaml
kubectl apply -f api-loadbalancer.yaml
```

Example usage after deploy
```bash
curl --header "Content-Type: application/json"  --request POST  --data '{"time": "2022-03-22T12:15:00.000Z","cookie": "foobar","country": "Poland","device": "PC", "action": "VIEW","origin": "test","product_info": {"product_id": "Siemens","brand_id": "Samsung","category_id": "SmartWatch","price": 420  } }'  -w "%{http_code}"  172.18.255.200:8080/user_tags 
```

