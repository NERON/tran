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
    <div id="main" style="width: 100%; min-height: 600px"></div>
    <script type="text/javascript">
	
		// based on prepared DOM, initialize echarts instance
		var myChart = echarts.init(document.getElementById('main'));

		$.getJSON("/chart/fgg/30", function(data) {
		
			var timestamps = data.map(function(val) {
				time = new Date(val.OpenTime);
				return time
			});
			
			var candlesticks = data.map(function(val) {
				return [val.OpenPrice,val.ClosePrice,val.LowPrice,val.HighPrice];
			});
			
			var scatter = data.map(function(val) {
				return val.IsRSIReverseLow ? val.LowPrice : NaN;
			});
			
		
			// specify chart configuration item and data
			var option = {
				animation: false,
				tooltip: {
					trigger: 'axis',
					axisPointer: {
						animation: false,
						type: 'cross',
						lineStyle: {
							width: 2,
							opacity: 1
						}
					}
				},
				xAxis: {
					data: timestamps,
					scale: true,
					axisPointer: {
						z: 100
					},
					axisLine: {
						onZero: false
					}
				},
				yAxis: [{
					scale: true,
					splitArea: {
						show: true
					},
					axisPointer: {
						z: 100
					}
				}],
				axisPointer: {
					link: {
						xAxisIndex: 'all'
					},
					label: {
						backgroundColor: '#777'
					}
				},
				dataZoom: [{ type: 'inside' }],
				series: [{
					name: 'cnd_data',
					type: 'k',
					data: candlesticks,
					itemStyle: {
						normal: {
							color0: '#FD1050',
							color: '#0CF49B',
							borderColor0: '#FD1050',
							borderColor: '#0CF49B'
						}
					}
				},
				{
					symbolSize: 5,
					data:scatter,
					type: 'scatter'
				}]
			};
			
			// use configuration item and data specified to show chart
			myChart.setOption(option);
			
			myChart.on('click',function(components){
				console.log(data[components.dataIndex]);
			})
		});
        
    </script>
</body>

</html>
