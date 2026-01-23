// Copy From https://github.com/jizizr/LanMei.git

extern crate wcloud as wc;
use image::{ImageBuffer, ImageError, ImageFormat, Luma, Rgba};
use lazy_static::lazy_static;
use rand::prelude::*;
use std::{
    collections::HashMap,
    fs::File,
    io::{Cursor, Read as _},
};
use wc::{Tokenizer, WordCloud, WordCloudSize};

lazy_static! {
    static ref WCLOUD: WordCloud = {
        let tokenizer = Tokenizer::default();
        WordCloud::default()
            .with_tokenizer(tokenizer)
            .with_word_rotate_chance(0.1)
            .with_background_color(Rgba([255, 255, 255, 255]))
            .with_font_from_path("./data/wcloud/font.ttf".into())
    };
    static ref MASK: ImageBuffer<Luma<u8>, Vec<u8>> = {
        let mut file = File::open("./data/wcloud/mask.png").expect("Unable to open mask file");
        let mut mask_buf = Vec::new();
        file.read_to_end(&mut mask_buf)
            .expect("Unable to read mask file");
        image::load_from_memory_with_format(&mask_buf, ImageFormat::Png)
            .expect("Unable to load mask from memory")
            .to_luma8()
    };
    static ref COLORS: [Rgba<u8>; 6] = [
        Rgba([16, 53, 123, 255]),   // #10357B
        Rgba([166, 190, 236, 255]), // #A6BEEC
        Rgba([108, 129, 176, 255]), // #6C81B0
        Rgba([9, 47, 112, 255]),    // #092F70
        Rgba([167, 170, 211, 255]), // #A7AAD3
        Rgba([117, 139, 196, 255]), // #758BC4
    ];
}

pub fn build(png_bytes: &mut Vec<u8>, words: HashMap<&str, usize>) -> Result<(), ImageError> {
    let start = std::time::Instant::now();
    let rgb_image = WCLOUD.generate_from_map_with_color_func(
        words,
        WordCloudSize::FromMask(MASK.clone()),
        5.0,
        |_, _| {
            let mut rng = rand::thread_rng();
            COLORS[rng.gen_range(0..COLORS.len())]
        },
    );
    rgb_image.write_to(&mut Cursor::new(png_bytes), ImageFormat::Png)?;
    println!("生成词云耗时: {:?}", start.elapsed());
    Ok(())
}
