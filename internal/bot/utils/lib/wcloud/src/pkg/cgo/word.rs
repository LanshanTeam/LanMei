// Copy From https://github.com/jizizr/LanMei.git

use std::{collections::HashMap, ffi::CStr};

#[repr(C)]
pub struct Word {
    word: *const libc::c_char,
    freq: usize,
}

pub fn process_words(words: *const Word, len: i32) -> HashMap<&'static str, usize> {
    let words_slice = unsafe {
        assert!(!words.is_null());
        std::slice::from_raw_parts(words, len as usize)
    };
    let mut h = HashMap::with_capacity(words_slice.len());
    for word in words_slice {
        let c_str = unsafe { CStr::from_ptr(word.word) };
        let rust_str = c_str.to_str().unwrap();
        h.insert(rust_str, word.freq);
    }
    h
}
