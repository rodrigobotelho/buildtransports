#!/bin/bash

#adiciona transporte
verifica_existencia_servico () {
    SERV=$1
    CRIADO=`find ${SERV}/pkg/service/ -name service.go 2>/dev/null`
    CRIADO2=`find ${SERV}/pkg/apis/service/ -name service.go 2>/dev/null`
    [[ "X${CRIADO}" = "X" ]] && [[ "X${CRIADO2}" = "X" ]] && return
}

verifica_module () {
    MODULE=$1
    FILE=$2
    CRIADO=`find pkg/${MODULE}/ -name ${FILE} 2>/dev/null`
    echo "find pkg/${MODULE}/ -name ${FILE} 2>/dev/null"
    if [ "X${CRIADO}" = "X" ] ; then
        return 0
    fi
    return 1
}

verifica_grpc () {
    verifica_module "grpc" "handler.go"
}

verifica_http () {
    verifica_module "http" "handler.go"
}

verifica_endpoint () {
    verifica_module "endpoint" "endpoint.go"
}

verifica_graphql () {
    cd $1
    verifica_module "apis/graphql" "handler.go"
    ret="$?"
    cd -
    return $ret
}

verifica_service() {
    verifica_module "service" "service.go"
}

move_module () {
    MODULE=$1
    
    mv pkg/${MODULE} pkg/apis/${MODULE}
    gomove pkg/${MODULE} pkg/apis/${MODULE}
}

move_service () {
    verifica_service 
    if [ "$?" -eq 1 ] ; then
        move_module "service"
    fi
}
move_grpc () {
    verifica_grpc
    if [ "$?" -eq 1 ] ; then
        move_module "grpc"
    fi
}

move_http () {
    verifica_http
    if [ "$?" -eq 1 ] ; then
        move_module "http"
    fi
}

move_endpoint () {
    verifica_endpoint
    if [ "$?" -eq 1 ] ; then
        move_module "endpoint"
    fi
}

if [ "$#" -ne 1 ] ; then 
    echo "Uso: $0 <nome de servico>"
    exit 1
fi


SERV=$1

if verifica_existencia_servico ${SERV} -eq 0 ; then
    kit n s ${SERV}
    echo "Adicione os métodos que serão utilizados no serviço: pkg/service/service.go"
    find ${SERV} |grep -v .git
    exit 1
fi

SERVNAME="$(tr '[:lower:]' '[:upper:]' <<< ${SERV:0:1})${SERV:1}"
HANDLER=${SERV}/pkg/apis/graphql/handler.go
HANDLER_TST=${SERV}/pkg/apis/graphql/handler_test.go
RESOLVER=${SERV}/pkg/apis/graphql/resolver.go
SCHEMA=${SERV}/pkg/apis/graphql/schema.graphql
SERVICE=${SERV}/cmd/service/service.go
PACKAGE=/src

while [ "X${TRANSPORT_DONE}" != "Xn" ] ; do
    echo "Indique qual transporte, http, grpc, graphql?"
    read TRANSP

    if [ "X${TRANSP}" = "Xgraphql" ]; then
        REWRITE_GRAPHQL="s"
        verifica_graphql ${SERV}
        if [ "$?" -eq 1 ] ; then
            echo "Transporte graphql já existente, TEM CERTEZA que deseja substituí-lo? s ou n?"
            read REWRITE_GRAPHQL
        fi
        if [ "X${REWRITE_GRAPHQL}" = "Xs" ] ; then
            mkdir -p ${SERV}/pkg/apis/graphql
            cat ${PACKAGE}/graphql/resolver.go|sed "s/Example/${SERVNAME}/g"|sed "s/example/${SERV}/g" > ${RESOLVER}
            goimports -w ${RESOLVER}
            cat ${PACKAGE}/graphql/handler.go|sed "s/Example/${SERVNAME}/g"|sed "s/example/${SERV}/g" > ${HANDLER}
            goimports -w ${HANDLER}
            cp ${PACKAGE}/graphql/handler_test.go ${HANDLER_TST}
            touch ${SCHEMA}
            cat ${PACKAGE}/graphql/init_handler.go|sed "s/Example/${SERVNAME}/g"|sed "s/example/${SERV}/g" >> ${SERVICE}
            cat ${SERVICE}|sed 's/var grpcAddr.*/&\nvar graphqlAddr = fs.String(\"graphql-addr\", \":8084\", \"graphql listen address\"\)/g' > ${SERVICE}_tmp
            mv ${SERVICE}_tmp ${SERVICE}
            goimports -w ${SERVICE}
        fi
    else
        echo "Indique os métodos separados por espaço, vazio se todos?"
        read METHODS

        if [ "X${METHODS}" != "X" ] ; then
            METHODS="-m ${METHODS}"
        fi

        if [ "X${TRANSP}" != "X" ] ; then
            TRANSP="-t ${TRANSP}"
        fi
        echo "kit g s ${SERV} ${TRANSP} ${METHODS}"
        kit g s ${SERV} ${TRANSP} ${METHODS}
        echo "kit g c ${SERV} ${TRANSP}"
        kit g c ${SERV} ${TRANSP}

    fi

    #colocando o init_handler
    cat ${PACKAGE}/init_service.go|sed "s/Example/${SERVNAME}/g" > ${SERV}/cmd/service/init_service.go
    cat ${SERVICE}|sed 's/svc := service.New.*/svc := initService()/g' > ${SERVICE}_tmp
    mv ${SERVICE}_tmp ${SERVICE}
    find ${SERV} |grep -v .git
    echo "Quer configurar um novo transporte? s ou n?"
    read TRANSPORT_DONE
done

#Corrigindo pastas
echo "Corrigindo pastas..."
cd ${SERV}

mkdir -p pkg/apis
move_service 
move_grpc
move_http
move_endpoint

goimports -w cmd/service/init_service.go
cd -
find ${SERV} |grep -v .git

