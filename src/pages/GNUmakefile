include ../../Rules.mak

all_local:
	hugo
	tar cfz tproxy.pages.tar.gz -C public .
	rm -rf public

clean_local:
	rm -f tproxy.pages.tar.gz

