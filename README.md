base setup
```
# global
$ ansible-galaxy install artis3n.tailscale

# pi
$ ansible-galaxy collection install community.docker
$ ansible-galaxy install mrlesmithjr.mdadm
$ ansible-galaxy install geerlingguy.nfs
$ ansible-galaxy install layereight.wifi

# k8s
$ ansible-galaxy install racqspace.microk8s
```

per pi
```
- 64 bit raspbian lite image
- configure in settings
    - enable ssh
    - key based auth
    - hostname so you can ssh in

now ssh pi@raspberrypi
```