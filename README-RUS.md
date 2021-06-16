<img src="/images/logo.png" alt="cTRL Logo"/>

cTRL это сервер написанный на языке Go, использующий <a href=https://github.com/eltaline/nutsdb>модифицированную</a> версию NutsDB базы, как бекенд для непрерывной очереди задач и сохранения результата исполнения команд из заданных задач в командных интерпретаторах типа /bin/bash на серверах, где данный сервис будет использоваться. При помощи cTRL можно получать через HTTP протокол задачи с командами для их исполнения на сервере и ограничивать количество одновременно исполняемых задач.

Текущая стабильная версия: 1.1.9
========

- <a href=/CHANGELOG-RUS.md>Changelog</a>

Несовместимости:

- Больше не поддерживается сборка под solaris/darwin из-за остутствия поддержки pgid

Исправлено в версии 1.1.9:

- Конвертация переменной

Возможности
========

- Многопоточность
- Поддержка HTTPS и Auth/IP авторизации
- Поддерживаемые HTTP методы: GET, POST
- Прием задач в очередь для отложенного параллельного исполнения команд
- Параллельное исполнение задач в реальном времени
- Ограничение максимального количества потоков на каждый виртуальный хост
- Поддерживает непрерывную очередь исполнения задач
- Автоматическое одновременное исполнение только одной одинаковой задачи
- Поддерживаемые интерпретаторы: /bin/bash , /bin/sh
- Поддерживаемые форматы: JSON

Требования
========

- Операционные системы: Linux, BSD, Solaris и OSX
- Архитектуры: AMD64, ARM64, PPC64 и MIPS64, проверялась только AMD64

Реальное применение
========

Мы используем данный сервер для исполнения отложенных задач по параллельной конвертации видео с помощью ffmpeg и ограничением процессов на нескольких серверах с Nvidia GPU, но сервер может исполнять любые команды через shell интерпретаторы.

Документация
========

Установка
--------

Установка пакетов/бинарников
--------

- <a href=https://github.com/eltaline/ctrl/releases>Скачать</a>

```
systemctl enable ctrl && systemctl start ctrl
```

Настройка и использование cTRL сервера
--------

Учетные данные по умолчанию: ```admin:swordfish```

Генерация пароля: ```echo -n "mypassword" | sha512sum```

В большинстве случаев достаточно воспользоваться конфигурационным файлом по умолчанию. Полное описание всех параметров продукта доступно здесь: <a href="/OPTIONS-RUS.md">Опции</a>

В данном руководстве используются идентификаторы типа UUID. Но клиент может устанавливать идентификаторы задач в произвольном формате.

Основные методы
--------

Запуск задачи или списка задач в реальном времени с ожиданием исполнения

```bash
curl -X POST -H "Auth: login:pass" "Content-Type: application/json" -d @task.json http://localhost/run
```

Постановка задачи или списка задач в очередь

```bash
curl -X POST -H "Auth: login:pass" -H "Content-Type: application/json" -d @task.json http://localhost/task
```

Получение задачи из очереди ожидания

```bash
curl -H "Auth: login:pass" "http://localhost/show?key=777a0d24-289e-4615-a439-0bd4efab6103&type=mytype&queue=received"
```

Получение всех задач из очереди ожидания

```bash
curl -H "Auth: login:pass" "http://localhost/show?queue=received"
```

Получение задачи из рабочей очереди

```bash
curl -H "Auth: login:pass" "http://localhost/show?key=777a0d24-289e-4615-a439-0bd4efab6103&type=mytype&queue=working"
```

Получение всех задач из рабочей очереди

```bash
curl -H "Auth: login:pass" "http://localhost/show?queue=working"
```

Получение задачи из списка завершенных задач

```bash
curl -H "Auth: login:pass" "http://localhost/show?key=777a0d24-289e-4615-a439-0bd4efab6103&type=mytype&queue=completed"
```

Получение всех задач из списка завершенных задач

```bash
curl -H "Auth: login:pass" "http://localhost/show?queue=completed"
```

Удаление задачи из очереди ожидания

```bash
curl -H "Auth: login:pass" "http://localhost/del?key=777a0d24-289e-4615-a439-0bd4efab6103&type=mytype&queue=received"
```

Удаление всех задач из очереди ожидания

```bash
curl -H "Auth: login:pass" "http://localhost/del?queue=received"
```

Удаление задачи из рабочей очереди

```bash
curl -H "Auth: login:pass" "http://localhost/del?key=777a0d24-289e-4615-a439-0bd4efab6103&type=mytype&queue=working"
```

Удаление всех задач из рабочей очереди

```bash
curl -H "Auth: login:pass" "http://localhost/del?queue=working"
```

Удаление задачи из списка завершенных задач

```bash
curl -H "Auth: login:pass" "http://localhost/del?key=777a0d24-289e-4615-a439-0bd4efab6103&type=mytype&queue=completed"
```

Удаление всех задач из списка завершенных задач

```bash
curl -H "Auth: login:pass" "http://localhost/del?queue=completed"
```

Формат ввода
--------

Описание полей
--------

- key - произвольный уникальный идентификатор (не допускается использовать двоеточие ":")
- type - произвольный тип задачи (не допускается использовать двоеточие ":")
- path - путь для смены директории перед запуском команды
- lock - произвольная метка блокировки
- command - команда
- threads - количество потоков для определенного типа задач
- timeout - таймаут
- ttltime - время жизни задачи в completed очереди
- interval - интервал между запуском задач
- repeaterr - перечисление ошибок, которые требуют повторного выполнения задачи
- repeatcnt - количество повторных запусков задачи в случае совпадения с любой ошибкой из параметра ```repeaterr```
- interr - перечисление ошибок, которые требуют снятия запущенной задачи
- intcnt - количество повторных запусков задачи в случае совпадения с любой ошибкой из параметра ```interr```
- lookout - включает или отключает реагирование на ошибки из stdout
- replace - перезапись одинаковой задачи с одинаковым ключем и одинаковым типом в очереди received

Поля ```threads/ttltime/interval/repeaterr/repeatcnt/interr/intcnt/lookout/replace``` актуальны только для задач установленных в очередь

Примеры установки задач
--------

Пример для одной задачи через /run:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","timeout":15}
]
```

Пример для списка задач через /run:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","timeout":15},
{"key":"4964deca-46ff-413f-8a92-e5baefd328e7","type":"mytype","path":"/","lock":"mylock2","command":"echo \"great\" && logger \"great\" && sleep 30","timeout":15},
{"key":"3fdf744d-36f1-499d-bd39-90a004ee39f6","type":"mytype","path":"/","lock":"mylock3","command":"echo \"world\" && logger \"world\" && sleep 15","timeout":15}
]
```

Пример для одной задачи через /task:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false}
]
```

Пример для списка задач через /task:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false},
{"key":"4964deca-46ff-413f-8a92-e5baefd328e7","type":"mytype","path":"/","lock":"mylock2","command":"echo \"great\" && logger \"great\" && sleep 30","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false},
{"key":"3fdf744d-36f1-499d-bd39-90a004ee39f6","type":"mytype","path":"/","lock":"mylock3","command":"echo \"world\" && logger \"world\" && sleep 15","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false}
]
```

Формат вывода
--------

Описание полей
--------

- key - произвольный уникальный идентификатор
- time - время приема задачи
- type - произвольный тип задачи
- path - путь для смены директории перед запуском команды
- lock - произвольная метка блокировки
- command - команда
- threads - количество потоков для определенного типа задач
- timeout - таймаут
- ttltime - время жизни задачи в completed очереди
- interval - интервал между запуском задач
- repeaterr - перечисление ошибок, которые требуют повторного выполнения задачи
- repeatcnt - количество повторных запусков задачи в случае совпадения с любой ошибкой из параметра ```repeaterr```
- interr - перечисление ошибок, которые требуют снятия запущенной задачи
- intcnt - количество повторных запусков задачи в случае совпадения с любой ошибкой из параметра ```interr```
- lookout - включает или отключает реагирование на ошибки из stdout
- replace - перезапись одинаковой задачи с одинаковым ключем и одинаковым типом в очереди received
- stdcode - не используется на данный момент
- stdout - стандартный вывод
- errcode - код ошибки
- stderr - стандартный вывод ошибки
- runtime - время выполнения в миллисекундах
- delcode - код ошибки при удалении задачи
- delerr - сообщение с ошибкой при удалении задачи

Примеры вывода
--------

Вывод завершенных задач:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","time":1589737139,"type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false,"stdcode":0,"stdout":"hello\n","errcode":0,"stderr":"","runtime":10010.669069},
{"key":"4964deca-46ff-413f-8a92-e5baefd328e7","time":1589737139,"type":"mytype","path":"/","lock":"mylock2","command":"echo \"great\" && logger \"great\" && sleep 30","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false,"stdcode":0,"stdout":"great\n","errcode":124,"stderr":"signal: killed","runtime":15006.034832},
{"key":"3fdf744d-36f1-499d-bd39-90a004ee39f6","time":1589737139,"type":"mytype","path":"/","lock":"mylock3","command":"echo \"world\" && logger \"world\" && sleep 15","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false,"stdcode":0,"stdout":"world\n","errcode":0,"stderr":"","runtime":15019.839685}
]
```

Вывод удаленных задач:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","time":1589737139,"type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false,"stdcode":0,"stdout":"hello\n","errcode":0,"stderr":"","runtime":10010.669069,"delcode":0,"delerr":""},
{"key":"4964deca-46ff-413f-8a92-e5baefd328e7","time":1589737139,"type":"mytype","path":"/","lock":"mylock2","command":"echo \"great\" && logger \"great\" && sleep 30","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false,"stdcode":0,"stdout":"great\n","errcode":124,"stderr":"signal: killed","runtime":15006.034832,"delcode":0,"delerr":""},
{"key":"3fdf744d-36f1-499d-bd39-90a004ee39f6","time":1589737139,"type":"mytype","path":"/","lock":"mylock3","command":"echo \"world\" && logger \"world\" && sleep 15","threads":4,"timeout":15,"ttltime":3600,"interval":1,"repeaterr":["CUDA_ERROR_OUT_OF_MEMORY","OtherError"],"repeatcnt":3,"interr":["Generic error in an external library","OtherError"],"intcnt":1,"lookout":false,"replace":false,"stdcode":0,"stdout":"world\n","errcode":0,"stderr":"","runtime":15019.839685,"delcode":0,"delerr":""}
]
```

Примечания и Q&A
--------

- В ключе и типе не допускается использовать двоеточие ":"
- Ограничение одновременно запущенных задач из очереди на каждый виртуальный хост регулируется посредством параметра vthreads в конфигурационном файле сервера или может быть переопределено через метод POST для каждого типа задачи
- Ограничение одновременно запущенных задач в реальном времени на каждый виртуальный хост регулируется посредством параметра rthreads в конфигурационном файле сервера
- Поле key, если данный идентификатор будет одинаковым для двух и более разных задач, в таком случае при выводе информации из очереди вы будете получать по данному идентификатору информацию сразу по нескольким задачам, это может быть полезно для группировки задач, но из очереди ожидания они будут выполняться в произвольном порядке
- Поля type и lock, если они будут назначены двум и более разным задачам абсолютно одинаковыми, в таком случае сервер будет выполнять эти задачи из очереди ожидания в произвольном порядке, но только поочереди
- Поля type и lock, установленные в реальном времени, не имеют значения, но они обязательны, все задачи будут выполняться по возможности в параллельном режиме.
- Для последовательного выполнения списка конкретных связанных друг с другом команд через очередь ожидания, устанавливайте данные команды в одну задачу, разделенные && или же напишите shell скрипт
- Задачи выполняемые в реальном времени исполняются параллельно.

Ошибки
--------

errcode
--------

- 0 (нет ошибки)
- 1 (любая ошибка)
- 124 (таймаут)
- 137 (переполнение памяти)
- 255 (не удалось запустить команду)

delcode
--------

- 0 (нет ошибки)
- 1 (любая ошибка)

ТуДу
========

- Любые предложения

Параметры
========

Полное описание всех параметров продукта доступно здесь: <a href="/OPTIONS-RUS.md">Опции</a>

HTTP Core
========

Использует <a href=https://github.com/kataras/iris>Iris</a> в качестве http ядра

Гарантии
========

Никакие гарантии на данное программное обеспечение не предоставляются. Просьба сначала тестировать.

Контакты
========

- Сайт компании: <a href="https://elta.ee">Eltaline</a>

```
Copyright © 2020 Andrey Kuvshinov. Contacts: <syslinux@protonmail.com>
Copyright © 2020 Eltaline OU. Contacts: <eltaline.ou@gmail.com>
All rights reserved.
```
