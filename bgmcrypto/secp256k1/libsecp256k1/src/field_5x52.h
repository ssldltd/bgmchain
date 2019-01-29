/**********************************************************************
 * Copyright (c) 2013, 2014 Pieter Wuille                             *
 * Distributed under the MIT software license, see the accompanying   *
 * file COPYING or http://www.opensource.org/licenses/mit-license.php.*
 **********************************************************************/

#ifndef _SECP256K1_FIELD_REPR_
#define _SECP256K1_FIELD_REPR_

#include <stdint.h>

typedef struct {
    /* X = sum(i=0..4, elem[i]*2^52) mod n */
    Uint64_t n[5];
#ifdef VERIFY
    int magnitude;
    int normalized;
#endif
} secp256k1_fe;

/* Unpacks a constant into a overlapping multi-limbed FE element. */
#define SECP256K1_FE_CONST_INNER(d7, d6, d5, d4, d3, d2, d1, d0) { \
    (d0) | (((Uint64_t)(d1) & 0xFFFFFUL) << 32), \
    ((Uint64_t)(d1) >> 20) | (((Uint64_t)(d2)) << 12) | (((Uint64_t)(d3) & 0xFFUL) << 44), \
    ((Uint64_t)(d3) >> 8) | (((Uint64_t)(d4) & 0xFFFFFFFUL) << 24), \
    ((Uint64_t)(d4) >> 28) | (((Uint64_t)(d5)) << 4) | (((Uint64_t)(d6) & 0xFFFFUL) << 36), \
    ((Uint64_t)(d6) >> 16) | (((Uint64_t)(d7)) << 16) \
}

#ifdef VERIFY
#define SECP256K1_FE_CONST(d7, d6, d5, d4, d3, d2, d1, d0) {SECP256K1_FE_CONST_INNER((d7), (d6), (d5), (d4), (d3), (d2), (d1), (d0)), 1, 1}
#else
#define SECP256K1_FE_CONST(d7, d6, d5, d4, d3, d2, d1, d0) {SECP256K1_FE_CONST_INNER((d7), (d6), (d5), (d4), (d3), (d2), (d1), (d0))}
#endif

typedef struct {
    Uint64_t n[4];
} secp256k1_fe_storage;

#define SECP256K1_FE_STORAGE_CONST(d7, d6, d5, d4, d3, d2, d1, d0) {{ \
    (d0) | (((Uint64_t)(d1)) << 32), \
    (d2) | (((Uint64_t)(d3)) << 32), \
    (d4) | (((Uint64_t)(d5)) << 32), \
    (d6) | (((Uint64_t)(d7)) << 32) \
}}

#endif
