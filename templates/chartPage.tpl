<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <!-- including ECharts file -->
    <script src="https://cdn.jsdelivr.net/npm/echarts@4.5.0/dist/echarts.min.js"></script>
	<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/3.4.1/jquery.min.js"></script>
	
</head>
<body>
    <!-- prepare a DOM container with width and height -->
    <div id="main" style="width: 800px;height:600px;"></div>
    <script type="text/javascript">
	
		// based on prepared DOM, initialize echarts instance
		var myChart = echarts.init(document.getElementById('main'));

		$.getJSON("/chart/fgg/30", function(data) {

			// specify chart configuration item and data
			var option = {
				title: {
					text: 'ECharts entry example'
				},
				tooltip: {},
				legend: {
					data: ['Sales']
				},
				xAxis: {
					data: ["shirt", "cardign", "chiffon shirt", "pants", "heels", "socks"]
				},
				yAxis: {},
				series: [{
					name: 'Sales',
					type: 'bar',
					data: [5, 20, 36, 10, 10, 20]
				}]
			};
			console.log(data);
			// use configuration item and data specified to show chart
			myChart.setOption(option);
		});
        
    </script>
</body>

</html>