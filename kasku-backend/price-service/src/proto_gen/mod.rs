// Module bridge ke kode hasil tonic-build.
//
// `build.rs` menjalankan tonic-build saat kompilasi dan menulis file
// `price.v1.rs` ke direktori `OUT_DIR` Cargo. Macro `tonic::include_proto!`
// di bawah memuat file tersebut ke dalam modul `price::v1`.
//
// Re-generate: cukup `cargo build` — protoc dijalankan otomatis via build.rs.
// Sumber `.proto`: `proto/price/v1/price.proto`.

#[allow(clippy::all)]
pub mod price {
    pub mod v1 {
        tonic::include_proto!("price.v1");
    }
}
