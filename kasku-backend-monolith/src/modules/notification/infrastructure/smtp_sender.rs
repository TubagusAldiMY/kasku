use lettre::{
    AsyncSmtpTransport, AsyncTransport, Message, Tokio1Executor,
    message::{Mailbox, MultiPart},
    transport::smtp::authentication::Credentials,
};

pub struct SmtpEmailSender {
    transport: AsyncSmtpTransport<Tokio1Executor>,
    from_address: String,
    from_name: String,
}

impl SmtpEmailSender {
    pub fn new(
        host: &str, port: u16,
        username: &str, password: &str,
        from_address: &str, from_name: &str,
    ) -> Result<Self, lettre::transport::smtp::Error> {
        let creds = Credentials::new(username.to_string(), password.to_string());
        let transport = AsyncSmtpTransport::<Tokio1Executor>::starttls_relay(host)?
            .port(port)
            .credentials(creds)
            .build();
        Ok(Self {
            transport,
            from_address: from_address.to_string(),
            from_name: from_name.to_string(),
        })
    }

    pub async fn send_html(&self, to_email: &str, subject: &str, html_body: &str) -> Result<(), String> {
        let from: Mailbox = format!("{} <{}>", self.from_name, self.from_address)
            .parse()
            .map_err(|e: lettre::address::AddressError| e.to_string())?;
        let to: Mailbox = to_email.parse()
            .map_err(|e: lettre::address::AddressError| e.to_string())?;

        let email = Message::builder()
            .from(from)
            .to(to)
            .subject(subject)
            .multipart(MultiPart::alternative_plain_html(
                "Email ini memerlukan klien email yang mendukung HTML.".to_string(),
                html_body.to_string(),
            ))
            .map_err(|e| e.to_string())?;

        self.transport.send(email).await.map_err(|e| e.to_string())?;
        Ok(())
    }
}
