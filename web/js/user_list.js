function get_user_list() {
  fetch(`/users_list_api`)
    .then((response) => response.json())
    .then((data) => {
      $("#myTable").DataTable({
        data: data[0].result,
        columns: [
          {
            data: null,
            render: function (_, _, row) {
              return row.firstName + " " + row.lastName;
            },
          },
          { data: "gender" },
          { data: "email" },
          { data: "contact" },
          { data: "position" },
          { data: "emergencyContact" },
          { data: "emergencyPersonRelation" },
        ],
      });
    })
    .catch((error) => {
      console.error("Error fetching records:", error);
    });
}

window.onload = get_user_list;
