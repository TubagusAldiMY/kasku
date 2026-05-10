fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(true)
        .build_client(false)
        .out_dir("src/proto_gen")
        .compile_protos(
            &["proto/price/v1/price.proto"],
            &["proto"],
        )?;
    Ok(())
}
