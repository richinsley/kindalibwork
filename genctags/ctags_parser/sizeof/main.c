#include <stdio.h>

  typedef enum {
    REQ_PAD_REFRESH = 1,
    REQ_PAD_UP,
    REQ_PAD_DOWN,
    REQ_PAD_LEFT,
    REQ_PAD_RIGHT,
    REQ_PAD_EXIT
  } Pad_Request;

int main() {
    printf("\'char\':%zu,\n", sizeof(char));
    printf("\'signed char\':%zu,\n", sizeof(signed char));
    printf("\'unsigned char\':%zu,\n", sizeof(unsigned char));
    printf("\'short\':%zu,\n", sizeof(short));
    printf("\'unsigned short\':%zu,\n", sizeof(unsigned short));
    printf("\'signed short\':%zu,\n", sizeof(signed short));
    printf("\'int\':%zu,\n", sizeof(int));
    printf("\'unsigned int\':%zu,\n", sizeof(unsigned int));
    printf("\'signed int\':%zu,\n", sizeof(signed int));
    printf("\'long\':%zu,\n", sizeof(long));
    printf("\'unsigned long\':%zu,\n", sizeof(unsigned long));
    printf("\'signed long\':%zu,\n", sizeof(signed long));
    printf("\'long long\':%zu,\n", sizeof(long long));
    printf("\'unsigned long long\':%zu,\n", sizeof(unsigned long long));
    printf("\'signed long long\':%zu,\n", sizeof(signed long long));
    printf("\'float\':%zu,\n", sizeof(float));
    printf("\'double\':%zu,\n", sizeof(double));
    printf("\'_Bool\':%zu,\n", sizeof(_Bool));
    printf("\'enum\':%zu,\n", sizeof(Pad_Request));
    return 0;
}