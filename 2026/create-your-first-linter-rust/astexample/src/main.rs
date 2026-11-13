const FOO: &str = "foo";

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct User {
    name: String,
    surname: String,
}

#[allow(dead_code)]
impl User {
    pub fn new(name: impl Into<String>, surname: impl Into<String>) -> Self {
        User {
            name: name.into(),
            surname: surname.into(),
        }
    }
}
