fn main() {
    let john = User::new("John", "Doe");
    println!("Hello, {:?}!", john);
}

#[derive(Debug)]
#[allow(dead_code)]
struct User {
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
