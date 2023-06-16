# frr-controller

## copy
```sh
cd frr-controller
cp ./frr-controller ./dist/images/frr-controller
```

## build git info
```sh
cd frr-controller/dist/images
./build_git_info.sh
```

## build image
```sh
cd frr-controller/dist/images
docker build -t nocsyscn/frr:0.1 -f Dockerfile .
```

## run frr controller
```sh
cd frr-controller/dist/yaml
kubectl apply -f frr-setup.yaml
kubectl apply -f frr.yaml
```