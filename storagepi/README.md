base setup
```
$ ansible-galaxy install artis3n.tailscale
$ ansible-galaxy collection install community.docker
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
