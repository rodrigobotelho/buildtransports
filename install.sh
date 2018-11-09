#Instala o pacote no path do go

echo "Instalando o buildtransports"

go get -u github.com/ksubedi/gomove
go get github.com/kujtimiihoxha/kit
go get golang.org/x/tools/cmd/goimports
go get -u google.golang.org/grpc
go get -u github.com/golang/protobuf/protoc-gen-go

echo "É preciso ter o protoc instalado no seu linux"

cp adiciona_transport.sh ${HOME}/go/bin/.
if [ ${PWD} != "${HOME}/go/src/github.com/rodrigobotelho/buildtransports" ] ;then
    mkdir -p ${HOME}/go/src/github.com/rodrigobotelho/buildtransports
    cp -r * ${HOME}/go/src/github.com/rodrigobotelho/buildtransports/.
fi

RUN=${HOME}/go/bin/adiciona_transport.sh
cat ${RUN} |sed 's/PACKAGE=.*/PACKAGE=${HOME}\/go\/src\/github.com\/rodrigobotelho\/buildtransports\/templates/g' > ${RUN}_tmp
mv ${RUN}_tmp ${RUN}

chmod 755 ${RUN}

echo "Pronto!"
