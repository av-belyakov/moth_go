Moth_go, version 1.22

Модифицирован раздел передачи файлов. Теперь одновременно через канал могут передоватся несколько файлов, а для того чтобы разделить их бинарные фрагменты используются специальные маркеры.

In home directory file .profile
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin