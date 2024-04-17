let userInfo = [];
// let email = "";

function updateUser(userData) {
  userInfo = userData;
  // console.log(userInfo);
}

// function deleteUser(email) {

//   delete_user_api(email);
// }

function delete_user_api(email) {

  Swal.fire({
    title: `Do you confirm delte user ${email}?`, showCancelButton: true, confirmButtonText: 'Yes',
  }).then((result) => {
    if (result.isConfirmed) {
      $.ajax({
        contentType: "application/json",
        type: "POST",
        url: "/delete_user_api",
        data: JSON.stringify({ "email": email }),
        success: function () {
          const Toast = Swal.mixin({
            toast: true,
            position: "top-end",
            showConfirmButton: false,
            timer: 3000,
            timerProgressBar: true,
            didOpen: (toast) => {
              toast.onmouseenter = Swal.stopTimer;
              toast.onmouseleave = Swal.resumeTimer;
            }
          });
          Toast.fire({
            icon: "success",
            title: "User has been deleted"
          });
          setTimeout(() => {
            location.reload();
          }
            , 3000);
        },
        error: function () {
          const Toast = Swal.mixin({
            toast: true,
            position: "top-end",
            showConfirmButton: false,
            timer: 3000,
            timerProgressBar: true,
            didOpen: (toast) => {
              toast.onmouseenter = Swal.stopTimer;
              toast.onmouseleave = Swal.resumeTimer;
            }
          });
          Toast.fire({
            icon: "error",
            title: "Delete user failed"
          });
        },
      });
    }
  })

}


function get_user_list() {
  fetch(`/users_list_api`)
    .then((response) => {
      if (response.status === 401) {
        window.location.replace("/?unauth=true")
        throw new Error("not login")
      }
      return response.json()
    })
    .then((data) => {
      for (let i = 0; i < data[0].result.length; i++) {
        data[0].result[i]["button"] = "View"
      }
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
          {
            'data': "button", render: function (_, _, row) {
              // console.log(row);
              return `<button data-ripple-light="true" data-dialog-target="edit-user-form" onclick='updateUser(["${row.email}","${row.firstName}","${row.lastName}","${row.gender}","${row.contact}","${row.position}","${row.emergencyContact}","${row.emergencyPersonRelation}"]);'
          class="select-none rounded-lg bg-gradient-to-tr from-gray-900 to-gray-800 py-3 px-6 text-center align-middle font-sans text-xs font-bold uppercase text-white shadow-md shadow-gray-900/10 transition-all hover:shadow-lg hover:shadow-gray-900/20 active:opacity-[0.85] disabled:pointer-events-none disabled:opacity-50 disabled:shadow-none">
          Edit
        </button>
        <button class='bg-red-500 ml-2 font-semibold p-2 rounded-md' onclick="delete_user_api('${row.email}')">Delete</button>`
            }
          }
        ],
      });
      fetch("/js/material-tailwind-dialog.js")
        .then(data => {
          return data.text()
        }).then(data => {
          eval(data)
        })
    })
    .catch((error) => {
      console.error("Error fetching records:", error);
    });
}
window.onload = get_user_list;
