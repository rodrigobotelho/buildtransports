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

verifica_apisgrpc () {
    verifica_module "apis/grpc" "handler.go"
}

verifica_apishttp () {
    verifica_module "apis/http" "handler.go"
}

verifica_apisendpoint () {
    verifica_module "apis/endpoint" "endpoint.go"
}

verifica_apisservice() {
    verifica_module "apis/service" "service.go"
}

move_module () {
    MODULE=$1

    mv pkg/${MODULE} pkg/apis/${MODULE}
    gomove pkg/${MODULE} pkg/apis/${MODULE}
}

move_back_module () {
    MODULE=$1

    mv pkg/apis/${MODULE} pkg/${MODULE}
    gomove pkg/apis/${MODULE} pkg/${MODULE}
}

move_service () {
    verifica_service 
    if [ "$?" -eq 1 ] ; then
        move_module "service"
    else
        verifica_apisservice
        if [ "$?" -eq 1 ] ; then
            move_back_module "service"
        fi
    fi
}

move_grpc () {
    verifica_grpc
    if [ "$?" -eq 1 ] ; then
        move_module "grpc"
    else
        verifica_apisgrpc
        if [ "$?" -eq 1 ] ; then
            move_back_module "grpc"
        fi
    fi
}

move_http () {
    verifica_http
    if [ "$?" -eq 1 ] ; then
        move_module "http"
    else
        verifica_apishttp
        if [ "$?" -eq 1 ] ; then
            move_back_module "http"
        fi
    fi
}

move_endpoint () {
    verifica_endpoint
    if [ "$?" -eq 1 ] ; then
        move_module "endpoint"
    else
        verifica_apisendpoint
        if [ "$?" -eq 1 ] ; then
            move_back_module "endpoint"
        fi
    fi
}

corrige_pastas () {
    cd ${SERV}
    if [ -d pkg/apis ] ; then
        move_service
        move_grpc
        move_http
        move_endpoint
    fi
    cd -
}

if [ "$#" -ne 1 ] ; then 
    echo "Uso: $0 <nome de servico>"
    exit 1
fi


SERV=$1

SERVICE=${SERV}/cmd/service/service.go
SERVNAME="$(tr '[:lower:]' '[:upper:]' <<< ${SERV:0:1})${SERV:1}"
HANDLER=${SERV}/pkg/apis/graphql/handler.go
HANDLER_TST=${SERV}/pkg/apis/graphql/handler_test.go
RESOLVER=${SERV}/pkg/apis/graphql/resolver.go
SCHEMA=${SERV}/pkg/apis/graphql/schema.graphql
HTTP_HANDLER=${SERV}/pkg/http/handler.go
PACKAGE=/src

if verifica_existencia_servico ${SERV} -eq 0 ; then
    kit n s ${SERV}
    #Coloca o pathprefix
    echo "//PathPrefix Prefixo do caminho do servico">> ${SERV}/pkg/service/service.go
    echo "const PathPrefix=\"\"" >> ${SERV}/pkg/service/service.go
    echo "Adicione os métodos que serão utilizados no serviço: pkg/service/service.go"
    find ${SERV} |grep -v .git
    exit 1
fi

corrige_pastas

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
            cat ${PACKAGE}/graphql/handler_test.go|sed "s/NewBasicExampleService/NewBasic${SERVNAME}Service/g"  > ${HANDLER_TST}
            touch ${SCHEMA}
            cat ${PACKAGE}/graphql/init_handler.go|sed "s/Example/${SERVNAME}/g"|sed "s/example/${SERV}/g" >> ${SERVICE}
            cat ${SERVICE}|sed 's/var grpcAddr.*/&\nvar graphqlAddr = fs.String(\"graphql-addr\", \":8084\", \"graphql listen address\"\)/g' > ${SERVICE}_tmp
            mv ${SERVICE}_tmp ${SERVICE}

            cat ${SERVICE}|sed 's/g := createService.*/&\n\tinitGraphqlHandler\(svc, g\)/g' > ${SERVICE}_tmp
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
        echo "kit g s ${SERV} --endpoint-mdw --svc-mdw ${TRANSP} ${METHODS}"
        kit g s ${SERV} --endpoint-mdw --svc-mdw ${TRANSP} ${METHODS}
        echo "kit g c ${SERV} ${TRANSP}"
        kit g c ${SERV} ${TRANSP}
        #Atualiza PathPrefix do http
        DEVE_ATUALIZAR=`cat ${HTTP_HANDLER} |grep "PathPrefix"`
        if [ "X${DEVE_ATUALIZAR}" == "X" ]; then
            cat ${HTTP_HANDLER} | sed 's/\(\"\/.*\"\)/service.PathPrefix + &/g' > ${HTTP_HANDLER}_tmp
            mv ${HTTP_HANDLER}_tmp ${HTTP_HANDLER}
            goimports -w  ${HTTP_HANDLER}
        fi

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
mkdir -p ${SERV}/pkg/apis
corrige_pastas
cd ${SERV}
goimports -w cmd/service/init_service.go
cd -
find ${SERV} |grep -v .git

