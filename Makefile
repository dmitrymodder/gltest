# Определение переменных
OUTDIR = build
TESTSDIR = tests
TESTOUTDIR = $(OUTDIR)/tests
# Устанавливаем GOFLAGS для подавления терминала в Windows
GOFLAGS = -ldflags "-H=windowsgui"

# Определение команды удаления в зависимости от ОС
ifeq ($(OS),Windows_NT)
    RM = if exist "$(OUTDIR)" rmdir /s /q "$(OUTDIR)"
else
    RM = rm -rf $(OUTDIR)
endif

# Все цели
all: clean dirs main tests send

# Создание необходимых директорий
dirs:
	if not exist "$(OUTDIR)" mkdir "$(OUTDIR)"
	if not exist "$(TESTOUTDIR)" mkdir "$(TESTOUTDIR)"

# Компиляция main.go в GLTest.exe
main: dirs
	go build $(GOFLAGS) -o $(OUTDIR)/GLTest.exe main.go

# Компиляция всех тестов
tests: dirs butterfly ocean triangles

# Компиляция butterfly.go
butterfly: dirs
	go build -o $(TESTOUTDIR)/butterfly.exe $(TESTSDIR)/butterfly.go

# Компиляция ocean.go
ocean: dirs
	go build -o $(TESTOUTDIR)/ocean.exe $(TESTSDIR)/ocean.go

# Компиляция triangles.go
triangles: dirs
	go build -o $(TESTOUTDIR)/triangles.exe $(TESTSDIR)/triangles.go

# Компиляция send.go в send.exe
send: dirs
	go build -ldflags "-H=windowsgui" -o $(OUTDIR)/send.exe send.go

# Очистка сборки
clean:
	$(RM)

# Цель .PHONY для команд, которые не создают файлы
.PHONY: all dirs clean tests butterfly ocean triangles send