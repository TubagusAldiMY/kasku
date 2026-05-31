use once_cell::sync::OnceCell;
use tera::Tera;

static TERA: OnceCell<Tera> = OnceCell::new();

pub fn init_templates(template_dir: &str) -> Result<(), tera::Error> {
    let pattern = format!("{}/**/*.html", template_dir);
    let tera = Tera::new(&pattern)?;
    let _ = TERA.set(tera);
    Ok(())
}

pub fn render(template_name: &str, ctx: &tera::Context) -> Result<String, tera::Error> {
    TERA.get()
        .expect("templates not initialized")
        .render(template_name, ctx)
}
