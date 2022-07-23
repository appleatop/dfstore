// Examples to initalize the mongdo DB 
db.dropDatabase('test')
db=db.getSiblingDB('dfstore1')
db.createCollection('table1')  
db.createCollection('table2')  
db.getCollectionNames()
db.createUser( { user: 'root',pwd: 'rootpass', roles: [ { role: "readWrite", db: "dfstore1"} ] } )
db.getUsers()