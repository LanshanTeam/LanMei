#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

typedef struct Word {
  const char *word;
  uintptr_t freq;
} Word;

int32_t add(int32_t a, int32_t b);

void free_string(char *s);

char *wcloud(const struct Word *words, int32_t len);
