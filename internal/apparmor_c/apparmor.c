#include "./apparmor.h"

int go_aa_change_hat(const char *hat, unsigned long magic)
{
    int ret = aa_change_hat(hat, magic);
    if (ret < 0)
    {
        return errno;
    }
    return 0;
}

int go_aa_change_profile(const char *profile)
{
    int ret = aa_change_profile(profile);
    if (ret < 0)
    {
        return errno;
    }
    return 0;
}

int go_aa_getcon(char **label, char **mode)
{
    int ret = aa_getcon(label, mode);
    if (ret < 0)
    {
        return errno;
    }
    return 0;
}

int go_aa_change_hatv(const char *hats[], unsigned long magic)
{
    int ret = aa_change_hatv(hats, magic);
    if (ret < 0)
    {
        return errno;
    }
    return 0;
}

int go_aa_getpeercon(int fd, char **label, char **mode)
{
    int ret = aa_getpeercon(fd, label, mode);
    if (ret < 0)
    {
        return errno;
    }
    return 0;
}