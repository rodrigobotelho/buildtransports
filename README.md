### Projeto para facilitar a criação transports usando o go-kit ###

Projeto pra adicionar parte de transporte em projetos do go, usando 
go-kit, kit-cli, go-imports, go-move

### Passos: ###
```
1- Cria um arquivo de service.go para adicionar os métodos do serviço
2- Deve alterar o arquivo service.go, adicionando os métodos
3- Adiciona o transport com 3 opções:
        * http
        * grpc
        * graphql
4- Os transports http e grpc pode indicar quais métodos estarão
   em cada um desses transports. Já o graphql não precisa, pois
   será indicado posteriormente nos arquivos de schema e resolver
5- http: Pronto para rodar
   grpc: 
        1- Deve alterar o *.proto colocando os campos do request e
           Reply
        2- Deve alterar os métodos enconde e decode do grpc/handler.go
           para fazer a tradução do pb para o serviço
        3- Deve alterar os métodos enconde e decode do client/grpc
           para fazer a tradução do serviço para o pb
   graphql:
        1- Deve alterar o schema.graphql para colocar os métodos e
           os tipos necessários 
        2- Deve alterar o resolver.go para implementar os métodos
           definidos no schema.graphql
```
