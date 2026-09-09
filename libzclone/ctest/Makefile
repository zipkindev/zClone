CFLAGS = -g -Wall

ifeq ($(OS),Windows_NT)
EXE = .exe
LIB = libzclone.lib
LDFLAGS = -L. -lzclone -lwinmm -lws2_32 -lole32
else
EXE =
LIB = libzclone.a
LDFLAGS = -L. -lzclone -lpthread -ldl
endif

ctest$(EXE): ctest.o $(LIB)
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

ctest.o: ctest.c libzclone.h
	$(CC) $(CFLAGS) -c $^ $(LDFLAGS)

$(LIB) libzclone.h:
	go build --buildmode=c-archive -o $(LIB) zclone/libzclone

test:	ctest$(EXE)
	./ctest$(EXE)

clean:
	rm -f tmp ctest ctest.exe *.o *.a *.lib *.h *.gch
