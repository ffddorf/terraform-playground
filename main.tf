resource "null_resource" "test" {

}

resource "random_string" "foo" {
  length = 20
}
