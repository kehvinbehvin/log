# Schemaless log parser design document

Production logs come in many shape and sizes which are dynamic over the lifetime of the production system.
Extracting data reliably from production logs requires planning and coordination and maintance. 
The thesis of this parser is that most logs can be broken down into top level regions delimited by symbols.
Symbols can be broken down into 2 types. Enclosing symbols and Non-enclosing symbols. Humans like to group
information together because it is easier to read. This is usually accomplished by enclosing symbols like
1. []
2. {}
3. ()
4. ""
5. ''
6. <>

Content within enclosing symbol can be primitive or non-primitive in nature which is outside the scope of
this parser.

This parser only aims to take a raw log line of any shape/type and reduce it into a signature of top level symbols
and the content of enclosing symbols to be replaced by the character 'X'


# Components
- Load a log file
- Read the file and process each line delimited by \n
- Batch update the output log file. (Balance .no of syscalls and memory)

# Line processor
- Use regex to remove all alphanumeric characters from line
- Use logic to remove all nested symbols in top-level symbols