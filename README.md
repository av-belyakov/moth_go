Application Moth_go, v1.39

Модифицирован раздел передачи файлов. Теперь одновременно через канал могут передоватся несколько файлов, а для того чтобы разделить их бинарные фрагменты используются специальные маркеры.

Исправлены ошибки с обработкой несуществующих директорий. Также 
теперь выполняется очистка срезов содержащих информацию по временному диапазону имеющихся файлов.

In home directory file .profile
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin