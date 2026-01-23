// Copy From https://github.com/jizizr/LanMei.git

mod pkg;
use base64::prelude::*;
use pkg::{cgo::word, wcloud};
use std::{collections::HashMap, ffi::CString, os::raw::c_char};

#[no_mangle]
pub extern "C" fn add(a: i32, b: i32) -> i32 {
    a + b
}

#[no_mangle]
pub extern "C" fn free_string(s: *mut c_char) {
    if s.is_null() {
        return;
    }
    unsafe {
        let _ = CString::from_raw(s);
    };
}

#[no_mangle]
pub extern "C" fn wcloud(words: *const word::Word, len: i32) -> *mut c_char {
    let h: HashMap<&str, usize> = word::process_words(words, len);
    let mut png: Vec<u8> = Vec::new();
    let _ = wcloud::gen::build(&mut png, h);
    let base_png = BASE64_STANDARD.encode(png);
    let c_string = CString::new(base_png).unwrap();
    c_string.into_raw()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn it_works() {
        let result = add(2, 2);
        assert_eq!(result, 4);
    }
}
