#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <sys/apparmor.h>

int go_aa_change_hat(const char *hat, unsigned long magic);

int go_aa_change_profile(const char *profile);

int go_aa_getcon(char **label, char **mode);

int go_aa_change_hatv(const char *hats[], unsigned long magic);

int go_aa_getpeercon(int fd, char **label, char **mode);
