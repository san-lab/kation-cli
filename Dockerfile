FROM golang:1.13

RUN printf "deb http://archive.debian.org/debian/ jessie main\ndeb-src http://archive.debian.org/debian/ jessie main\ndeb http://security.debian.org jessie/updates main\ndeb-src http://security.debian.org jessie/updates main" > /etc/apt/sources.list

RUN apt-get update && apt-get install -y \
        curl \
        sudo \
        wget


# Install a recent version of nodejs
RUN curl -sL https://deb.nodesource.com/setup_10.x | sudo bash - && sudo apt-get install -y nodejs
COPY . /go/src/github.com/clearmatics/ion-cli

# Install the current compatible solc version
RUN wget https://github.com/ethereum/solidity/releases/download/v0.5.12/solc-static-linux -O solc
RUN chmod +x ./solc
RUN cp ./solc /go/src/github.com/clearmatics/ion-cli
ENV PATH $PATH:/go/src/github.com/clearmatics/ion-cli

WORKDIR /go/src/github.com/clearmatics/ion-cli

CMD ["/bin/bash"]
