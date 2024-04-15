const image = new Image();

window.onload = async function () {
  await get_warning_day_count();
  await get_warning_month_record();
  await get_warning_year_count();
  $("#date-time-now").text(new Date().toLocaleString());
};

let warningData = new Array(12).fill(0);
let violationCount = new Array(3).fill(0);




async function get_warning_day_count() {
  const response = await fetch(`/get_warning_day_count_api`);
  const data = await response.json();
  $("#warning_day_count").text(data[0].result[0].count);
}

async function get_warning_month_record() {
  try {
    const response = await fetch(`/get_violation_month_record_api`);
    const data = await response.json();
    for (let i = 0; i < data[0].result.length; i++) {
      for (let j = 0; j < data[0].result[i].violation_type.length; j++) {
        if (data[0].result[i].violation_type[j] == "NO_SAFETY_VEST") {
          violationCount[0] += 1;
        } else if (data[0].result[i].violation_type[j] == "NO_MASK") {
          violationCount[1] += 1;
        } else if (data[0].result[i].violation_type[j] == "NO_HARDHAT") {
          violationCount[2] += 1;
        }
      }
    }

    new Chart(document.getElementById("common violation"), {
      type: "doughnut",
      plugins: [
        {
          id: "customCanvasBackgroundImage",
          beforeDraw: (chart) => {
            if (image.complete) {
              const ctx = chart.ctx;
              const { top, left, width, height } = chart.chartArea;
              const x = left + width / 2 - image.width / 2;
              const y = top + height / 2 - image.height / 2;
              ctx.drawImage(image, x, y);
            } else {
              image.onload = () => chart.draw();
            }
          },
        },
      ],
      data: {
        labels: ["No Safety Vest", "No Mask", "No Hardhat"],
        datasets: [
          {
            label: "Violation Record",
            data: violationCount,
            backgroundColor: [
              "rgb(255, 99, 132)",
              "rgb(54, 162, 235)",
              "rgb(255, 205, 86)",
            ],
            hoverOffset: 4,
          },
        ],
      },
      options: {
        plugins: {
          legend: {
            labels: {
              color: 'white',
              font: {
                size: 16
              }
            }
          },
          tooltip: {
            bodyFont: {
              size: 20,
              color: 'white'
            },
            titleFont: {
              size: 20,
              color: 'white'
            }
          }
        }
      },
    });

    // console.log(violationCount);
  } catch (error) {
    console.error("Error:", error);
  }
}



async function get_warning_year_count() {
  // results = await get_warning_year_count()
  response = await fetch(`/get_warning_year_count_api`);
  results = await response.json();
  for (let i = 0; i < results[0].result.length; i++) {
    warningData[results[0].result[i]['month'] - 1] = results[0].result[i]['count'];
  }

  new Chart(document.getElementById("warning"), {
    type: "line",
    data: {
      labels: [
        "Jan", "Feb", "Mar", "Apr", "May", "Jun",
        "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
      ],
      datasets: [
        {
          label: "Num of warning",
          data: warningData,
          backgroundColor: "yellow",
          borderWidth: 2,
          borderColor: "yellow",
        },
      ],
    },
    options: {
      responsive: true,
      scales: {
        x: {
          ticks: {
            font: {
              size: 20 // Set font size here for x-axis labels
            },
            color: 'white'
          }
        },
        y: {
          ticks: {
            font: {
              size: 20 // Set font size here for y-axis labels
            },
            color: 'white'
          }
        }
      },
      plugins: {
        legend: {
          labels: {
            font: {
              size: 20 // Set font size for legend labels
            },
            color: 'white'
          }
        }
      }
    },
  });
}
