# Recon My Way. 

###  Tools and scripts setting up guide for personal use. 

## Important things to Install before setting up tools (Debian Based OS)

### Git Installation

```bash
root@recon:~# sudo apt-get upgrade
root@recon:~# sudo apt-get update
root@recon:~# sudo apt-get install git
```

### Curl  installation. 

```bash
root@recon:~# apt install curl
```

### Go language installation. 

```bash
root@recon:~# curl -O https://dl.google.com/go/go1.10.2.linux-amd64.tar.gz
root@recon:~# sha256sum go1.10.2-linux-amd64.tar.gz
root@recon:~# tar xvf go1.10.2.linux-amd64.tar.gz
root@recon:~# sudo chown -R root:root ./go
root@recon:~# sudo mv go /usr/local
root@recon:~# vi ~/.profile
```

#### and add the following lines in `.profile`

```bash
export GOPATH=$HOME/work
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
source ~/.profile
```

#### Cleaing Up

```bash
root@recon:~# rm -rf go1.10.1.linux-amd64.tar.gz
root@recon:~# rm -rf work
```

### Ruby Language installation. 

```bash
root@recon:~# apt-get install ruby-full
```

### Pip & pip3 install.

```bash
root@recon:~# apt install python-pip
root@recon:~# apt install python3-pip	//for python 3
```

## Setting up tools for subdomain.rb & recon.rb. 

## subdomain.rb

### colorize gem install

```bash
root@recon:~# gem install colorize
```


### Amass

```
root@recon:~# go get -u github.com/OWASP/Amass/...
```



### Subfinder

The installation is easy. You can download the pre-built binaries for different platforms from the [Releases](https://github.com/projectdiscovery/subfinder/releases/) page. Extract them using tar, move it to your $PATH and you're ready to go.

```bash
> tar -xzvf subfinder-linux-amd64.tar
> mv subfinder-linux-amd64 /usr/bin/subfinder
> subfinder 
```

### Subdomains by censys

```bash
root@recon:~# pip install censys
root@recon:~# export CENSYS_API_ID=Your_Censys_APP_ID
root@recon:~# export CENSYS_API_SECRET=Your_Censys_APP_ID
```





