db.dropDatabase('test')
db.getSiblingDB('dfstore1')
db.createCollection('table1')  
db.createCollection('table2')  
db.getCollectionNames()
db.createUser( { user: 'root',pwd: 'rootpass', roles: [ { role: "readWrite", db: "dfstore1"} ] } )
db.getUsers()